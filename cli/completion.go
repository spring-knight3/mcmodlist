package cli

import (
	"fmt"
	"strings"
)

func completionScript(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "bash":
		return bashCompletion(), nil
	case "zsh":
		return zshCompletion(), nil
	case "fish":
		return fishCompletion(), nil
	case "powershell", "pwsh":
		return powershellCompletion(), nil
	default:
		return "", fmt.Errorf("unknown shell %q (want bash, zsh, fish, or powershell)", shell)
	}
}

const completionWords = `--instance --format --output --force --no-summary --summary-only --no-header --color --no-color --verbose --quiet --help --version --launcher --list-instances --include-disabled --include-nested --recursive --jobs --columns --sort --group-by --wide --truncate --copy --search --loader --side --author --unrecognised --check --diff completion bash zsh fish powershell pretty markdown csv json html text auto always never`

func bashCompletion() string {
	return `# mcmodlist bash completion
_mcmodlist() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  case "$prev" in
    --format|-f) COMPREPLY=( $(compgen -W "pretty markdown csv json html text" -- "$cur") ); return 0 ;;
    --color) COMPREPLY=( $(compgen -W "auto always never" -- "$cur") ); return 0 ;;
    --group-by) COMPREPLY=( $(compgen -W "loader side" -- "$cur") ); return 0 ;;
    --launcher) COMPREPLY=( $(compgen -W "prism multimc curseforge modrinth atlauncher gdlauncher" -- "$cur") ); return 0 ;;
    --sort) COMPREPLY=( $(compgen -W "name id version author jar loader side status" -- "$cur") ); return 0 ;;
    completion) COMPREPLY=( $(compgen -W "bash zsh fish powershell" -- "$cur") ); return 0 ;;
  esac
  COMPREPLY=( $(compgen -W "` + completionWords + `" -- "$cur") )
}
complete -F _mcmodlist mcmodlist
`
}

func zshCompletion() string {
	return `#compdef mcmodlist
_mcmodlist() {
  local -a formats colors groups launchers shells sorts
  formats=(pretty markdown csv json html text)
  colors=(auto always never)
  groups=(loader side)
  launchers=(prism multimc curseforge modrinth atlauncher gdlauncher)
  shells=(bash zsh fish powershell)
  sorts=(name id version author jar loader side status)
  _arguments \
    '--instance[instance directory]:path:_files -/' \
    '--format[output format]:format:'$formats \
    '--output[write output to file]:file:_files' \
    '--force[overwrite an existing output file]' \
    '--no-summary[omit the instance summary]' \
    '--summary-only[print only the instance summary]' \
    '--no-header[omit headers for data-only output]' \
    '--color[color mode]:color:'$colors \
    '--no-color[disable colour]' \
    '--verbose[print extra diagnostics to stderr]' \
    '--quiet[suppress warnings]' \
    '--help[show help]' \
    '--version[show version]' \
    '--launcher[filter --list-instances by launcher]:launcher:'$launchers \
    '--list-instances[list known instances and exit]' \
    '--include-disabled[include disabled mods]' \
    '--include-nested[include mods nested inside jars]' \
    '--recursive[scan subdirectories of the mods folder]' \
    '--jobs[jar scan worker count]:number:' \
    '--columns[comma-separated columns]:columns:' \
    '--sort[sort spec]:sort:'$sorts \
    '--group-by[group output]:group:'$groups \
    '--wide[do not truncate table columns]' \
    '--truncate[truncate table cells]:number:' \
    '--copy[copy rendered output to the clipboard]' \
    '--search[filter mods by text]:text:' \
    '--loader[filter mods by loader]:loader:' \
    '--side[filter mods by side]:side:' \
    '--author[filter mods by author]:text:' \
    '--unrecognised[show only unrecognised jars]' \
    '--check[check for problems and exit 3 if found]' \
    '--diff[compare with another instance]:instance:_files -/' \
    '1:command:(completion)' \
    '*::arg:->args'
  case $state in
    args) _arguments '--shell[shell]:shell:'$shells ;;
  esac
}
_mcmodlist "$@"
`
}

func fishCompletion() string {
	return `# mcmodlist fish completion
complete -c mcmodlist -l instance -s i -d 'instance directory'
complete -c mcmodlist -l format -s f -d 'output format' -a 'pretty markdown csv json html text'
complete -c mcmodlist -l output -s o -d 'write output to a file'
complete -c mcmodlist -l force -d 'overwrite an existing output file'
complete -c mcmodlist -l no-summary -d 'omit the instance summary'
complete -c mcmodlist -l summary-only -d 'print only the instance summary'
complete -c mcmodlist -l no-header -d 'omit headers for data-only output'
complete -c mcmodlist -l color -d 'color mode' -a 'auto always never'
complete -c mcmodlist -l no-color -d 'disable colour'
complete -c mcmodlist -l verbose -s v -d 'print extra diagnostics to stderr'
complete -c mcmodlist -l quiet -s q -d 'suppress warnings'
complete -c mcmodlist -l help -s h -d 'show help'
complete -c mcmodlist -l version -s V -d 'show version'
complete -c mcmodlist -l launcher -d 'filter --list-instances by launcher' -a 'prism multimc curseforge modrinth atlauncher gdlauncher'
complete -c mcmodlist -l list-instances -d 'list known instances and exit'
complete -c mcmodlist -l include-disabled -d 'include disabled mods'
complete -c mcmodlist -l include-nested -d 'include mods nested inside jars'
complete -c mcmodlist -l recursive -d 'scan subdirectories of the mods folder'
complete -c mcmodlist -l jobs -d 'jar scan worker count'
complete -c mcmodlist -l columns -d 'comma-separated columns'
complete -c mcmodlist -l sort -d 'sort spec' -a 'name id version author jar loader side status'
complete -c mcmodlist -l group-by -d 'group output' -a 'loader side'
complete -c mcmodlist -l wide -d 'do not truncate table columns'
complete -c mcmodlist -l truncate -d 'truncate table cells'
complete -c mcmodlist -l copy -d 'copy rendered output to the clipboard'
complete -c mcmodlist -l search -d 'filter mods by text'
complete -c mcmodlist -l loader -d 'filter mods by loader' -a 'fabric quilt forge neoforge connector'
complete -c mcmodlist -l side -d 'filter mods by side' -a 'client server both'
complete -c mcmodlist -l author -d 'filter mods by author'
complete -c mcmodlist -l unrecognised -d 'show only unrecognised jars'
complete -c mcmodlist -l check -d 'check for problems and exit 3 if found'
complete -c mcmodlist -l diff -d 'compare with another instance'
complete -c mcmodlist -l completion -a 'bash zsh fish powershell' -d 'print shell completion'
`
}

func powershellCompletion() string {
	return `# mcmodlist powershell completion
Register-ArgumentCompleter -Native -CommandName mcmodlist -ScriptBlock {
  param($wordToComplete, $commandAst, $cursorPosition)
  $flags = @(
    '--instance','--format','--output','--force','--no-summary','--summary-only','--no-header','--color','--no-color',
    '--verbose','--quiet','--help','--version','--launcher','--list-instances','--include-disabled','--include-nested',
    '--recursive','--jobs','--columns','--sort','--group-by','--wide','--truncate','--copy','--search','--loader',
    '--side','--author','--unrecognised','--check','--diff','completion'
  )
  $flags | Where-Object { $_ -like "$wordToComplete*" }
}
`
}
