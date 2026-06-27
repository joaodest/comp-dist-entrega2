#requires -Version 5.1
<#
.SYNOPSIS
  Compila o relatorio da Entrega 1 (entrega1.tex -> entrega1.pdf) e valida o
  limite de 4 paginas exigido pelo enunciado.

.DESCRIPTION
  Usa latexmk (preferencial) ou pdflatex. Se nenhuma toolchain LaTeX estiver
  disponivel, falha com instrucoes de instalacao em vez de gerar nada
  silenciosamente. Quando pdfinfo (poppler/MiKTeX) esta presente, verifica que
  o PDF gerado tem no maximo 4 paginas.
#>

$ErrorActionPreference = 'Stop'
Set-Location -Path $PSScriptRoot

$tex = 'entrega1.tex'
$pdf = 'entrega1.pdf'

function Test-Command([string]$name) {
    return [bool](Get-Command $name -ErrorAction SilentlyContinue)
}

# 1. Detecta a toolchain LaTeX.
$hasLatexmk  = Test-Command 'latexmk'
$hasPdflatex = Test-Command 'pdflatex'

if (-not ($hasLatexmk -or $hasPdflatex)) {
    Write-Error @"
Nenhuma toolchain LaTeX encontrada (latexmk ou pdflatex).

Instale uma das opcoes abaixo e rode novamente:
  - MiKTeX (Windows):  https://miktex.org/download
  - TeX Live (multiplataforma): https://tug.org/texlive/

Apos instalar, garanta que 'pdflatex' esteja no PATH e execute:
  pwsh docs/report/build.ps1
"@
    exit 1
}

# 2. Compila.
if ($hasLatexmk) {
    Write-Host '==> Compilando com latexmk...'
    & latexmk -pdf -interaction=nonstopmode -halt-on-error $tex
    if ($LASTEXITCODE -ne 0) { Write-Error "latexmk falhou (exit $LASTEXITCODE)."; exit 1 }
} else {
    Write-Host '==> Compilando com pdflatex (duas passadas)...'
    & pdflatex -interaction=nonstopmode -halt-on-error $tex
    if ($LASTEXITCODE -ne 0) { Write-Error "pdflatex falhou (exit $LASTEXITCODE)."; exit 1 }
    & pdflatex -interaction=nonstopmode -halt-on-error $tex
    if ($LASTEXITCODE -ne 0) { Write-Error "pdflatex falhou na 2a passada (exit $LASTEXITCODE)."; exit 1 }
}

# 3. Verifica que o PDF existe.
if (-not (Test-Path $pdf)) {
    Write-Error "Build terminou mas '$pdf' nao foi gerado."
    exit 1
}
Write-Host "==> PDF gerado: $((Resolve-Path $pdf).Path)"

# 4. Valida o limite de 4 paginas (quando pdfinfo estiver disponivel).
if (Test-Command 'pdfinfo') {
    $pages = (& pdfinfo $pdf | Select-String -Pattern '^Pages:\s+(\d+)').Matches.Groups[1].Value
    if ($pages) {
        Write-Host "==> Paginas: $pages"
        if ([int]$pages -gt 4) {
            Write-Error "O relatorio tem $pages paginas; o limite da Entrega 1 e 4 paginas."
            exit 1
        }
    }
} else {
    Write-Host '==> pdfinfo nao encontrado: pulei a verificacao do limite de 4 paginas.'
}

Write-Host '==> OK.'
