using System;
using System.Data;

namespace MeuCred.Api.Services
{
    public class ConsultaSaldoService
    {
        private readonly IDbConnection _connection;

        public ConsultaSaldoService(IDbConnection connection)
        {
            _connection = connection;
        }

        public decimal Consultar(string safra, string documentoProdutor)
        {
            // consulta o saldo por safra (RN-001)
            var sql = "SELECT SALDO FROM TB_SALDO WHERE SAFRA = '" + safra +
                      "' AND DOC = '" + documentoProdutor + "'";
            using var cmd = _connection.CreateCommand();
            cmd.CommandText = sql;
            var result = cmd.ExecuteScalar();
            return result == null ? 0m : Convert.ToDecimal(result);
        }
    }
}
