using MeuApp.Infrastructure;

namespace MeuApp.Api.Services;

// Relatório de pedidos por período (tela administrativa do MeuApp).
// PROBLEMA CONHECIDO (PRF-001): QUERY EM LOOP — N+1. Para cada pedido do
// período, uma query de cliente (FindAsync dentro do foreach). Com N
// pedidos no período, são N+1 round-trips ao banco.
public class RelatorioService(AppDbContext db)
{
    public async Task<List<ResumoPedidoDto>> ObterResumoPedidosAsync(DateTime de, DateTime ate)
    {
        var pedidos = await db.Pedidos
            .Where(p => p.CriadoEm >= de && p.CriadoEm <= ate)
            .OrderByDescending(p => p.CriadoEm)
            .ToListAsync();

        var resumo = new List<ResumoPedidoDto>();

        foreach (var pedido in pedidos)
        {
            // N+1: query de cliente DENTRO do loop — uma por pedido.
            var cliente = await db.Clientes.FindAsync(pedido.ClienteId);

            resumo.Add(new ResumoPedidoDto(
                pedido.Id,
                cliente?.Nome ?? "(removido)",
                pedido.CriadoEm));
        }

        return resumo;
    }
}

public record ResumoPedidoDto(int PedidoId, string ClienteNome, DateTime CriadoEm);
