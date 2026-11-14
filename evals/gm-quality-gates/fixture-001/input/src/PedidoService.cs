using MeuApp.Api.Domain;

namespace MeuApp.Api.Services;

public class PedidoService
{
    private readonly IPedidoRepository _pedidos;
    private readonly IClienteRepository _clientes;

    public PedidoService(IPedidoRepository pedidos, IClienteRepository clientes)
    {
        _pedidos = pedidos;
        _clientes = clientes;
    }

    public decimal CalcularTotal(IReadOnlyList<ItemPedido> itens)
    {
        decimal total = 0m;
        foreach (var item in itens)
        {
            total += item.Quantidade * item.PrecoUnitario;
        }
        return total;
    }

    public async Task<Resultado> RegistrarAsync(NovoPedido comando)
    {
        var cliente = await _clientes.ObterAsync(comando.ClienteId);
        if (cliente is null)
        {
            return Resultado.Falha("CLIENTE_INEXISTENTE");
        }

        var pedido = new Pedido(cliente.Id, comando.Itens, CalcularTotal(comando.Itens));
        await _pedidos.InserirAsync(pedido);
        return Resultado.Ok(pedido.Id);
    }
}
