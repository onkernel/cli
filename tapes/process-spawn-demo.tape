Output tapes/process-spawn-demo.gif
Output tapes/process-spawn-demo.mp4

Require kernel

Set Shell "bash"
Set FontSize 32
Set Width 2400
Set Height 1200
Set TypingSpeed 100ms

Type "kernel browsers create -H -p process-spawn-demo" Enter
Sleep 2s

Type "PROC_ID=$(kernel --no-color browsers process spawn process-spawn-demo -- /bin/cat | awk '/Process ID/ {print $NF}')" Enter
Sleep 4s
Type "echo Process ID: $PROC_ID" Enter
Sleep 2s

Type "kernel browsers process status process-spawn-demo $PROC_ID" Enter
Sleep 2s

Type "DATA=$(printf 'hello from stdin\n' | base64)" Enter
Sleep 2s
Type "kernel browsers process stdin process-spawn-demo $PROC_ID --data-b64 $DATA" Enter
Sleep 2s

Type "kernel browsers process stdout-stream process-spawn-demo $PROC_ID" Enter
Sleep 3s
Ctrl+C
Sleep 1s

Type "kernel browsers process kill process-spawn-demo $PROC_ID" Enter
Sleep 2s
Type "kernel browsers process status process-spawn-demo $PROC_ID" Enter
Sleep 2s
Type "kernel browsers delete -y process-spawn-demo" Enter
Sleep 5s


