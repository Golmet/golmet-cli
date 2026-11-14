// Código legado fictício da fixture (MeuApp) — domínio Estoque
public class EstoqueService
{
    public void Reserva(ItemPedido item)
    {
        item.ReservaExpiraEm = DateTime.Now.AddMinutes(30); // reserva 30 min
    }
}
