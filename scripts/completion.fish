# Copyright Stealth Scale B.V. 2026
# SPDX-License-Identifier: MIT

# fish completion of {{app}}.
# Load it with: {{app}} completion fish | source
# or write it to ~/.config/fish/completions/{{app}}.fish.

function __{{fn}}_complete
    set -l tokens (commandline --current-process --tokenize --cut-at-cursor)
    set -l current (commandline --current-token --cut-at-cursor)
    set -l program (string replace -r -- '^~' $HOME $tokens[1])
    set -e tokens[1]
    set -l lines ($program completion fish -- $tokens "$current" 2>/dev/null)
    set -q lines[1]; or return
    set -l directive $lines[-1]
    set -e lines[-1]
    switch $directive
        case :values
            for line in $lines
                echo $line
            end
        case ':files*'
            set -l exts (string split -n ' ' -- (string replace -- :files '' $directive))
            set -l prefix (string match -r -- '^-[^=]*=' "$current")
            set -l path (string replace -r -- '^-[^=]*=' '' "$current")
            for file in (__fish_complete_path "$path")
                set -l name (string split -f1 \t -- $file)
                if not set -q exts[1]; or test -d "$name"
                    echo "$prefix$file"
                    continue
                end
                for ext in $exts
                    if string match -q -- "*.$ext" $name
                        echo "$prefix$file"
                        break
                    end
                end
            end
    end
end

complete -c {{app}} -f -a '(__{{fn}}_complete)'
