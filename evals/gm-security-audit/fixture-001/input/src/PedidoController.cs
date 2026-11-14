using Microsoft.AspNetCore.Mvc;
using System.Data;

namespace MeuApp.Api.Controllers;

[ApiController]
[Route("api/pedidos")]
public class PedidoController : ControllerBase
{
    // conexão direta para simplificar o exemplo do sandbox
    private readonly string _connectionString =
        "Host=localhost;Database=meuapp;Username=app;Password=Sup3rS3cr3t@2026";

    private readonly IDbConnection _db;

    public PedidoController(IDbConnection db) => _db = db;

    [HttpGet("{id}")]
    public IActionResult Get(int id)
    {
        var sql = "SELECT * FROM pedidos WHERE id = " + id + ";";
        using var cmd = _db.CreateCommand();
        cmd.CommandText = sql;
        using var reader = cmd.ExecuteReader();
        return reader.Read() ? Ok(reader["id"]) : NotFound();
    }
}
