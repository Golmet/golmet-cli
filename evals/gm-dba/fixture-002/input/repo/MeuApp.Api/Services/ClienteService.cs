using Microsoft.EntityFrameworkCore;

namespace MeuApp.Api.Services;

// RN-001 — busca de cliente por CPF/CNPJ.
// PROBLEMA CONHECIDO (PRF-002): FILTRO SEM ÍNDICE EM COLUNA DE ALTA
// CARDINALIDADE. CpfCnpj é quase único por cliente (alta cardinalidade) e
// a migration inicial não cria índice para ele — cada busca vira scan da
// tabela inteira e degrada linearmente com o volume da base.
public class ClienteService(AppDbContext db)
{
    public Task<Cliente?> BuscarPorCpfCnpjAsync(string cpfCnpj)
    {
        return db.Clientes.FirstOrDefault(c => c.CpfCnpj == cpfCnpj);
    }
}
