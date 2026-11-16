#!/usr/bin/env node
// golmet.js — o launcher npm do golmet (M31 — a pendência da M4): detecta
// a plataforma (process.platform + process.arch), resolve o binário
// nativo do sub-pacote @golmet/cli-<os>-<arch> (instalado pelas
// optionalDependencies — uma falha graciosamente no npm) e o despacha com
// stdio inherit — o exit code do binário propaga (o contrato do CLI).
// Sem dependências; Node >=14.
//
// O -v/--version delega ao binário (todos os args passam direto — o
// launcher não interpreta nada).
'use strict';

const { spawn } = require('child_process');

// platformPackage — a convenção dos sub-pacotes de plataforma.
function platformPackage(platform, arch) {
  return '@golmet/cli-' + platform + '-' + arch;
}

// binaryName — o nome do binário dentro do sub-pacote.
function binaryName(platform) {
  return 'golmet' + (platform === 'win32' ? '.exe' : '');
}

// resolveTarget — resolve o binário nativo relativo ao launcher (os
// optionalDependencies são irmãos dele no node_modules).
function resolveTarget(pkgName, binName) {
  return require.resolve(pkgName + '/' + binName);
}

function missingPlatformMessage(platform, arch) {
  return 'golmet: o binário da sua plataforma (' + platform + '/' + arch +
    ') não acompanhou a instalação — reinstale com `npm install -g golmet-cli`' +
    ' ou baixe manualmente em github.com/Golmet/golmet-cli/releases';
}

function main() {
  const pkg = platformPackage(process.platform, process.arch);
  let target;
  try {
    target = resolveTarget(pkg, binaryName(process.platform));
  } catch (err) {
    // o optionalDependencies falhou graciosamente no npm (SO fora da
    // matriz, npm antigo sem suporte a optional, instalação parcial) —
    // erro claro com os dois caminhos
    console.error(missingPlatformMessage(process.platform, process.arch));
    process.exit(2);
  }
  const child = spawn(target, process.argv.slice(2), { stdio: 'inherit' });
  child.on('error', (err) => {
    console.error('golmet: falha ao executar o binário (' + target + '): ' + err.message);
    process.exit(2);
  });
  child.on('close', (code) => {
    process.exit(code == null ? 1 : code);
  });
}

// exportado para os testes (o main roda só quando executado direto)
module.exports = { platformPackage, binaryName, resolveTarget, missingPlatformMessage };
if (require.main === module) {
  main();
}
