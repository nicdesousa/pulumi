using System.Threading.Tasks;
using Grpc.Core;
using Pulumirpc;

namespace Pulumi
{
    internal interface IMonitor
    {
        Task<InvokeResponse> InvokeAsync(InvokeRequest request);

        Task<ReadResourceResponse> ReadResourceAsync(Resource resource, ReadResourceRequest request);

        Task<RegisterResourceResponse> RegisterResourceAsync(Resource resource, RegisterResourceRequest request);

        Task RegisterResourceOutputsAsync(RegisterResourceOutputsRequest request);

        Task LogAsync(LogRequest request);

        Task<SetRootResourceResponse> SetRootResourceAsync(SetRootResourceRequest request);

        Task<GetRootResourceResponse> GetRootResourceAsync(GetRootResourceRequest request);
    }
    
    class GrpcMonitor : IMonitor
    {
        private readonly ResourceMonitor.ResourceMonitorClient _client;
        private readonly Engine.EngineClient _engine;

        public GrpcMonitor(string engine, string monitor)
        {
            this._engine = new Engine.EngineClient(new Channel(engine, ChannelCredentials.Insecure));
            this._client = new ResourceMonitor.ResourceMonitorClient(new Channel(monitor, ChannelCredentials.Insecure));
        }

        public async Task<InvokeResponse> InvokeAsync(InvokeRequest request)
            => await this._client.InvokeAsync(request);

        public async Task<ReadResourceResponse> ReadResourceAsync(Resource resource, ReadResourceRequest request)
            => await this._client.ReadResourceAsync(request);

        public async Task<RegisterResourceResponse> RegisterResourceAsync(Resource resource, RegisterResourceRequest request)
            => await this._client.RegisterResourceAsync(request);

        public async Task RegisterResourceOutputsAsync(RegisterResourceOutputsRequest request)
            => await this._client.RegisterResourceOutputsAsync(request);

        public async Task LogAsync(LogRequest request)
            => await this._engine.LogAsync(request);

        public async Task<SetRootResourceResponse> SetRootResourceAsync(SetRootResourceRequest request)
            => await this._engine.SetRootResourceAsync(request);

        public async Task<GetRootResourceResponse> GetRootResourceAsync(GetRootResourceRequest request)
            => await this._engine.GetRootResourceAsync(request);
    }
}
