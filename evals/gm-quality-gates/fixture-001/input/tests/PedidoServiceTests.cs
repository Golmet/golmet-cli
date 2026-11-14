using MeuApp.Api.Domain;
using MeuApp.Api.Services;
using Xunit;

namespace MeuApp.Api.Tests;

public class PedidoServiceTests
{
    [Fact]
    public void CalcularTotal_WithTwoItems_ReturnsSum()
    {
        // Arrange
        var itens = new List<ItemPedido>
        {
            new("SKU-1", 2, 10m),
            new("SKU-2", 1, 5m),
        };

        // Act
        var service = new PedidoService(null!, null!);
        var total = service.CalcularTotal(itens);

        // Assert
        Assert.Equal(25m, total);
    }

    [Fact]
    public async Task Registrar_WithUnknownClient_ReturnsFailure()
    {
        // Arrange
        var clientes = new ClienteRepositoryFake(clienteExistente: false);
        var service = new PedidoService(new PedidoRepositoryFake(), clientes);

        // Act
        var resultado = await service.RegistrarAsync(NovoPedidoFake.Valido());

        // Assert
        Assert.False(resultado.Sucesso);
        Assert.Equal("CLIENTE_INEXISTENTE", resultado.Erro);
    }
}
