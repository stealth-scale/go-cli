# Copyright Stealth Scale B.V. 2026
# SPDX-License-Identifier: MIT

# bash completion of {{app}}, for bash 4.1 or later.
# Load it with: source <({{app}} completion bash)

__{{fn}}_complete() {
    local line=${COMP_LINE:0:COMP_POINT} breaks=${COMP_WORDBREAKS:-$' \t\n'}
    local cur=${line##*["$breaks"]}
    local -a words lines exts
    read -r -a words <<<"$line"
    if [[ -z $line || $line == *[[:space:]] ]]; then
        words+=("")
    fi
    local program=${words[0]} word=${words[${#words[@]}-1]}
    if [[ $program == "~"* ]]; then
        program=$HOME${program:1}
    fi
    mapfile -t lines < <("$program" completion bash -- "${words[@]:1}" 2>/dev/null)
    local count=${#lines[@]}
    if ((count == 0)); then
        return 0
    fi
    local directive=${lines[count - 1]} candidate file ext
    COMPREPLY=()
    case $directive in
    :values)
        local strip=${word%"$cur"}
        for candidate in "${lines[@]:0:count-1}"; do
            COMPREPLY+=("${candidate#"$strip"}")
        done
        ;;
    :files*)
        read -r -a exts <<<"${directive#:files}"
        compopt -o filenames 2>/dev/null
        while IFS= read -r file; do
            if ((${#exts[@]} == 0)) || [[ -d $file ]]; then
                COMPREPLY+=("$file")
                continue
            fi
            for ext in "${exts[@]}"; do
                if [[ $file == *."$ext" ]]; then
                    COMPREPLY+=("$file")
                    break
                fi
            done
        done < <(compgen -f -- "$cur")
        ;;
    esac
}

complete -F __{{fn}}_complete {{app}}
