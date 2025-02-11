# VHS documentation
#
# Output:
#   Output <path>.gif               Create a GIF output at the given <path>
#   Output <path>.mp4               Create an MP4 output at the given <path>
#   Output <path>.webm              Create a WebM output at the given <path>
#
# Require:
#   Require <string>                Ensure a program is on the $PATH to proceed
#
# Settings:
#   Set FontSize <number>           Set the font size of the terminal
#   Set FontFamily <string>         Set the font family of the terminal
#   Set Height <number>             Set the height of the terminal
#   Set Width <number>              Set the width of the terminal
#   Set LetterSpacing <float>       Set the font letter spacing (tracking)
#   Set LineHeight <float>          Set the font line height
#   Set LoopOffset <float>%         Set the starting frame offset for the GIF loop
#   Set Theme <json|string>         Set the theme of the terminal
#   Set Padding <number>            Set the padding of the terminal
#   Set Framerate <number>          Set the framerate of the recording
#   Set PlaybackSpeed <float>       Set the playback speed of the recording
#   Set MarginFill <file|#000000>   Set the file or color the margin will be filled with.
#   Set Margin <number>             Set the size of the margin. Has no effect if MarginFill isn't set.
#   Set BorderRadius <number>       Set terminal border radius, in pixels.
#   Set WindowBar <string>          Set window bar type. (one of: Rings, RingsRight, Colorful, ColorfulRight)
#   Set WindowBarSize <number>      Set window bar size, in pixels. Default is 40.
#   Set TypingSpeed <time>          Set the typing speed of the terminal. Default is 50ms.
#
# Sleep:
#   Sleep <time>                    Sleep for a set amount of <time> in seconds
#
# Type:
#   Type[@<time>] "<characters>"    Type <characters> into the terminal with a
#                                   <time> delay between each character
#
# Keys:
#   Escape[@<time>] [number]        Press the Escape key
#   Backspace[@<time>] [number]     Press the Backspace key
#   Delete[@<time>] [number]        Press the Delete key
#   Insert[@<time>] [number]        Press the Insert key
#   Down[@<time>] [number]          Press the Down key
#   Enter[@<time>] [number]         Press the Enter key
#   Space[@<time>] [number]         Press the Space key
#   Tab[@<time>] [number]           Press the Tab key
#   Left[@<time>] [number]          Press the Left Arrow key
#   Right[@<time>] [number]         Press the Right Arrow key
#   Up[@<time>] [number]            Press the Up Arrow key
#   Down[@<time>] [number]          Press the Down Arrow key
#   PageUp[@<time>] [number]        Press the Page Up key
#   PageDown[@<time>] [number]      Press the Page Down key
#   Ctrl+<key>                      Press the Control key + <key> (e.g. Ctrl+C)
#
# Display:
#   Hide                            Hide the subsequent commands from the output
#   Show                            Show the subsequent commands in the output

Output demo.gif

Require echo

Set Margin 20
Set MarginFill "#674EFF"
Set BorderRadius 15
Set Padding 20
Set WindowBar Colorful
Set Shell "zsh"
Set FontSize 20
Set FontFamily "Victor Mono"
Set Width 1200
Set Height 700

Hide
Type "cd ~/Scratch/Projects" Enter
Type "source ~/.zshrc" Enter
Type "yahh remove" Enter
Type "yahh create" Enter
Type "clear" Enter
Sleep 2
Type "cat banner.txt" Sleep 500ms  Enter
Show
Sleep 5s
Type "echo 'This is the global history realm.'" Sleep 500ms  Enter
Type "echo 'The history has only these commands so far.'" Sleep 500ms  Enter
Type "history" Sleep 500ms Enter
Sleep 3s
Type "> 'Now, let me create my first project, called Project-1'" Sleep 1s Ctrl+u
Sleep 1
Type "mkdir Project-1" Sleep 500ms Enter
Sleep 1
Type "cd Project-1" Sleep 500ms Enter
Type "> 'Creating a new history realm inside Project-1...'" Sleep 1s Ctrl+u
Type "yahh create" Sleep 500ms Enter
Sleep 3
Type "> 'Let me check the history inside Project-1...'" Sleep 1s Ctrl+u
Type "history" Sleep 500ms Enter
Sleep 2s
Type "> 'It is empty!'" Sleep 2s Ctrl+u
Type "echo This is a command inside Project-1" Sleep 500ms Enter
Type "> 'And the history now contains...'" Sleep 1s Ctrl+u
Type "history" Sleep 500ms Enter
Type "> 'It does not matter how deep inside Project-1 I am...'" Sleep 1s Ctrl+u
Type "mkdir sub" Sleep 500ms Enter
Type "cd sub" Sleep 500ms Enter
Type "mkdir dir" Sleep 500ms Enter
Type "cd dir" Sleep 500ms Enter
Type "history" Sleep 500ms Enter
Sleep 1s
Type "> 'They all share the same Project-1 history.'" Sleep 2s Ctrl+u
Type "cd .." Sleep 500ms Enter
Sleep 500ms
Type "cd .." Sleep 500ms Enter
Sleep 500ms
Type "> 'But if I go outside the Project-1 directory...'" Sleep 2s Ctrl+u
Type "cd .." Sleep 500ms Enter
Sleep 1s
Type "history" Sleep 500ms Enter
Type "> '...then I start to see my global history again!'" Sleep 2s Ctrl+u
Sleep 1s
Type "mkdir Project-2" Sleep 500ms Enter
Sleep 500ms
Type "cd Project-2" Sleep 500ms Enter
Sleep 500ms
Type "history" Sleep 500ms Enter
Type "> 'Project-2 still sees the global history, until a new realm is created...'" Sleep 2s Ctrl+u
Type "yahh create" Sleep 500ms Enter
Sleep 3
Type "history" Sleep 500ms Enter
Sleep 1
Type "> 'The right realm will be activated automatically upon changing projects.'" Sleep 2s Ctrl+u
Sleep 500ms
Type "cd ../Project-1/sub/dir" Sleep 500ms Enter
Sleep 500ms
Type "history" Sleep 500ms Enter
Sleep 3
Type "cd -" Sleep 500ms Enter
Sleep 500ms
Type "history" Sleep 500ms Enter
Type "> 'Thanks for watching!'" Sleep 5s Ctrl+u
Hide
Type "cd ~/Scratch/Projects" Enter
Type "cd Project-1" Enter
Type "yahh remove" Enter
Type "cd ~/Scratch/Projects" Enter
Type "rm -rf Project-1" Enter
Type "cd Project-2" Enter
Type "yahh remove" Enter
Type "cd ~/Scratch/Projects" Enter
Type "rm -rf Project-2" Enter
Type "yahh remove" Enter
