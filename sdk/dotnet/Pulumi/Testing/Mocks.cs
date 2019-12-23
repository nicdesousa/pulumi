using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Linq;
using System.Threading.Tasks;
using Google.Protobuf.Collections;
using Google.Protobuf.WellKnownTypes;
using Pulumi.Serialization;
using Pulumirpc;

namespace Pulumi.Testing
{
    /// <summary>
    /// Hooks to mock the engine and provide test doubles for offline unit testing of stacks.
    /// </summary>
    public interface IMocks
    {
        /// <summary>
        /// Invoked when a new resource is created by the program.
        /// </summary>
        /// <param name="type">Resource type name.</param>
        /// <param name="name">Resource name.</param>
        /// <param name="inputs">Dictionary of resource input properties.</param>
        /// <param name="provider">Provider.</param>
        /// <param name="id">Resource identifier.</param>
        /// <returns>A tuple of a resource identifier and resource state. State can be either a POCO
        /// or a dictionary bag.</returns>
        Task<(string id, object state)> NewResourceAsync(string type, string name,
            ImmutableDictionary<string, object> inputs, string? provider, string? id);

        /// <summary>
        /// Invoked when the program needs to call a provider to load data (e.g., to retrieve an existing
        /// resource).
        /// </summary>
        /// <param name="token">Function token.</param>
        /// <param name="args">Dictionary of input arguments.</param>
        /// <param name="provider">Provider.</param>
        /// <returns>Invocation result, can be either a POCO or a dictionary bag.</returns>
        Task<object> CallAsync(string token, ImmutableDictionary<string, object> args, string? provider);
    }
    
    internal class DefaultMocks : IMocks
    {
        public Task<(string id, object state)> NewResourceAsync(string type, string name,
            ImmutableDictionary<string, object> inputs, string? provider, string? id)
        {
            var outputs = inputs.Add("name", "test");
            return Task.FromResult((type + "-test", (object)outputs));

        }

        public Task<object> CallAsync(string token, ImmutableDictionary<string, object> args, string? provider)
            => Task.FromResult((object)args); // We may want something smarter here, I haven't got to invokes yet.
    }

    internal class MockMonitor : IMonitor
    {
        private readonly IMocks _mocks;
        private readonly Serializer _serializer = new Serializer();
        private string? _rootResourceUrn = null;
        
        public readonly List<string> Errors = new List<string>();
        public readonly List<Resource> Resources = new List<Resource>();

        public ImmutableDictionary<string, object> StackOutputs { get; private set; } =
            ImmutableDictionary<string, object>.Empty;

        public MockMonitor(IMocks mocks)
        {
            _mocks = mocks;
        }

        public async Task<InvokeResponse> InvokeAsync(InvokeRequest request)
        {
            var result = await _mocks.CallAsync(request.Tok, ToDictionary(request.Args), request.Provider);
            return new InvokeResponse { Return = await SerializeAsync(result) };
        }

        public async Task<ReadResourceResponse> ReadResourceAsync(Resource resource, ReadResourceRequest request)
        {
            var (id, state) = await _mocks.NewResourceAsync(request.Type, request.Name,
                ToDictionary(request.Properties), request.Provider, request.Id);
            this.Resources.Add(resource);
            return new ReadResourceResponse
            {
                Urn = NewUrn(request.Parent, request.Type, request.Name),
                Properties = await SerializeAsync(state)
            };
        }

        public async Task<RegisterResourceResponse> RegisterResourceAsync(Resource resource, RegisterResourceRequest request)
        {
            var (id, state) = await _mocks.NewResourceAsync(request.Type, request.Name, ToDictionary(request.Object),
                request.Provider, request.ImportId);
            this.Resources.Add(resource);
            return new RegisterResourceResponse
            {
                Id = id ?? request.ImportId,
                Urn = NewUrn(request.Parent, request.Type, request.Name),
                Object = await SerializeAsync(state)
            };
        }

        public Task RegisterResourceOutputsAsync(RegisterResourceOutputsRequest request)
        {
            var stackUrn = $"urn:pulumi:{Deployment.Instance.StackName}::{Deployment.Instance.ProjectName}::{Stack._rootPulumiStackTypeName}::{Deployment.Instance.ProjectName}-{Deployment.Instance.StackName}";
            if (request.Urn == stackUrn)
            {
                StackOutputs = ToDictionary(request.Outputs);
            }

            return Task.CompletedTask;
        }

        public Task LogAsync(LogRequest request)
        {
            if (request.Severity == LogSeverity.Error)
            {
                this.Errors.Add(request.Message);
            }
            
            return Task.CompletedTask;
        }

        public Task<SetRootResourceResponse> SetRootResourceAsync(SetRootResourceRequest request)
        {
            _rootResourceUrn = request.Urn;
            return Task.FromResult(new SetRootResourceResponse());
        }

        public Task<GetRootResourceResponse> GetRootResourceAsync(
            GetRootResourceRequest request)
            => Task.FromResult(new GetRootResourceResponse { Urn = _rootResourceUrn });
        
        private static string NewUrn(string parent, string type, string name)
        {
            if (!string.IsNullOrEmpty(parent)) 
            {
                var qualifiedType = parent.Split("::")[2];
                var parentType = qualifiedType.Split("$").First();
                type = parentType + "$" + type;
            }
            return "urn:pulumi:" + string.Join("::", new[] { Deployment.Instance.StackName, Deployment.Instance.ProjectName, type, name });
        }

        private static ImmutableDictionary<string, object> ToDictionary(Struct s)
        {
            var builder = ImmutableDictionary.CreateBuilder<string, object>();
            foreach (var (key, value) in s.Fields)
            {
                var data = Deserializer.Deserialize(value);
                if (data.IsKnown && data.Value != null)
                {
                    builder.Add(key, data.Value);
                }
            }
            return builder.ToImmutable();
        }

        private async Task<Struct> SerializeAsync(object o)
        {
            var dict = (o as IDictionary<string, object>)?.ToImmutableDictionary()
                   ?? await _serializer.SerializeAsync("", o) as ImmutableDictionary<string, object>
                   ?? ImmutableDictionary<string, object>.Empty;
            return Serializer.CreateStruct(dict);
        }
    }

    /// <summary>
    /// Represents an outcome of a test run.
    /// </summary>
    public class TestResult
    {
        /// <summary>
        /// Whether the test run failed with an error.
        /// </summary>
        public bool HasErrors { get; }

        /// <summary>
        /// Error messages that were logged during the run.
        /// </summary>
        public ImmutableArray<string> LoggedErrors { get; }

        /// <summary>
        /// All Pulumi resources that got registered during the run.
        /// </summary>
        public ImmutableArray<Resource> Resources { get; }
        
        public ImmutableDictionary<string, object> StackOutputs { get; }

        // TODO: this is an awkward method that I had to add to extract values from outputs. Is there a better way?
        public Task<T> GetAsync<T>(Output<T> output) => output.GetValueAsync();

        internal TestResult(bool hasErrors, IEnumerable<string> loggedErrors, IEnumerable<Resource> resources, ImmutableDictionary<string, object> stackOutputs) 
        {
            this.HasErrors = hasErrors;
            this.LoggedErrors = loggedErrors.ToImmutableArray();
            this.Resources = resources.ToImmutableArray();
            this.StackOutputs = stackOutputs;
        }
    }
}
