#compdef {{app}}
# zsh completion of {{app}}.
# Load it with: source <({{app}} completion zsh)
# or write it to a file _{{app}} in a directory of $fpath.

_{{fn}}() {
    local -a args lines candidates exts
    local program=${(Q)words[1]} line value text directive
    if [[ $program == '~'* ]]; then
        program=$HOME${program#'~'}
    fi
    args=("${(@Q)words[2,CURRENT-1]}" "${(Q)PREFIX}")
    lines=("${(@f)$("$program" completion zsh -- "${args[@]}" 2>/dev/null)}")
    directive=${lines[-1]}
    case $directive in
    :values)
        for line in "${(@)lines[1,-2]}"; do
            value=${line%%$'\t'*}
            text=${line#*$'\t'}
            if [[ "$text" == "$line" ]]; then
                text=
            fi
            candidates+=("${value//:/\\:}${text:+:$text}")
        done
        _describe -t values value candidates
        ;;
    :files*)
        if [[ $PREFIX == -*=* ]]; then
            compset -P '*='
        fi
        exts=(${=directive#:files})
        if ((${#exts})); then
            _files -g "*.(${(j:|:)exts})"
        else
            _files
        fi
        ;;
    esac
}

if [[ $funcstack[1] == _{{app}} ]]; then
    _{{fn}} "$@"
else
    compdef _{{fn}} {{app}}
fi
