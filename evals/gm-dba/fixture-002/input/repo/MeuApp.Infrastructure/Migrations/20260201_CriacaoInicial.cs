// MeuApp — migration inicial (hand-written, playbook dotnet — EF Core).
// Fixture mínima: o que importa para o serviço performance do gm-dba é o
// que EXISTE de índice aqui — PKs de identidade + UNIQUE Email (Clientes)
// + UNIQUE Sku (Produtos). NÃO existe índice em Clientes.CpfCnpj nem em
// Pedidos (ClienteId, CriadoEm) — lacunas apontadas no DB_PERFORMANCE.md.
using System;
using Microsoft.EntityFrameworkCore.Migrations;

#nullable disable

namespace MeuApp.Infrastructure.Migrations
{
    public partial class CriacaoInicial : Migration
    {
        protected override void Up(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.CreateTable(
                name: "Clientes",
                columns: table => new
                {
                    Id = table.Column<int>(nullable: false)
                        .Annotation("SqlServer:Identity", "1, 1"),
                    Nome = table.Column<string>(maxLength: 200, nullable: false),
                    Email = table.Column<string>(maxLength: 200, nullable: false),
                    CpfCnpj = table.Column<string>(maxLength: 20, nullable: false),
                    CriadoEm = table.Column<DateTime>(nullable: false)
                },
                constraints: table =>
                {
                    table.PrimaryKey("PK_Clientes", x => x.Id);
                });

            // Pedidos / ItensPedidos / Produtos: criação análoga (fixture
            // mínima — detalhe de colunas irrelevante para os achados).

            migrationBuilder.CreateIndex(
                name: "IX_Clientes_Email",
                table: "Clientes",
                column: "Email",
                unique: true);

            migrationBuilder.CreateIndex(
                name: "IX_Produtos_Sku",
                table: "Produtos",
                column: "Sku",
                unique: true);
        }

        protected override void Down(MigrationBuilder migrationBuilder)
        {
            migrationBuilder.DropTable(name: "Clientes");
        }
    }
}
