using System.Threading.Tasks;
using Pulumirpc;

namespace Pulumi
{
    /// <summary>
    /// Abstraction for hooks to the engine and monitor.
    /// </summary>
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
}
