---
project: MEUAPP
module: seguranca
status: draft
---

# Segurança — MeuApp-API

## Requisitos específicos do módulo

- Segredos exclusivamente via variável de ambiente ou secret manager;
  nenhum literal de credencial no código-fonte.
- Hash de token e senha com primitiva dedicada (SHA-256/HMAC ou
  algoritmo de hash de senha); MD5 e SHA-1 proibidos.
- Queries de banco sempre parametrizadas; concatenação de input em SQL
  é violação bloqueante.

Conciliação com a base T1 security-review: conflito entre este documento
e a base é sinalizado no relatório de auditoria, não decidido na skill.
