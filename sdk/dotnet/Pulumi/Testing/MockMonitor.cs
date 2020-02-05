using System.Collections.Generic;
using System.Collections.Immutable;
using System.Linq;
using System.Threading.Tasks;
using Google.Protobuf.WellKnownTypes;
using Pulumi.Serialization;
using Pulumirpc;

namespace Pulumi.Testing
{
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
            var stackUrn = NewUrn("", Stack._rootPulumiStackTypeName, $"{Deployment.Instance.ProjectName}-{Deployment.Instance.StackName}");
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
}
