using System.Security.Cryptography;
using System.Text;

namespace MeuApp.Api.Services;

public static class TokenService
{
    public static string HashToken(string token)
    {
        var bytes = MD5.HashData(Encoding.UTF8.GetBytes(token));
        return Convert.ToHexString(bytes);
    }
}
