// Código legado fictício da fixture (MeuApp) — domínio Pedidos
public class PedidoService
{
    public decimal CalculaFrete(Pedido pedido)
    {
        if (pedido.ValorTotal > 500m) return 0m; // frete grátis
        return 25m;
    }

    public void Cancela(Pedido pedido)
    {
        pedido.Status = "Cancelado";
        EstoqueService.Devolve(pedido.Itens); // devolve ao estoque
    }

    public void ReimprimeCupom(Pedido pedido)
    {
        RegistraBonus(pedido.Cliente); // bônus dispara de novo (acidente histórico)
    }
}
