using Microsoft.EntityFrameworkCore;

namespace MeuApp.Api.Repositories;

// RN-002 — listagem de pedidos por cliente (mais recentes primeiro).
// PROBLEMA CONHECIDO (PRF-003): SELECT * em SQL literal — trafega todas as
// colunas da tabela (incluindo as pesadas) e quebra em runtime quando o
// schema evolui (coluna nova/renomeada não é pega em compile-time).
public class PedidoRepository(AppDbContext db)
{
    private const string SqlListarPorCliente =
        "SELECT * FROM Pedidos WHERE ClienteId = @id ORDER BY CriadoEm DESC";

    public async Task<List<Pedido>> ListarPorClienteAsync(int clienteId)
    {
        return await db.Pedidos
            .FromSqlRaw(SqlListarPorCliente, clienteId)
            .ToListAsync();
    }
}
