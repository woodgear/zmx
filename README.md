# zmx: bring M-x experience to zsh
/zmacs/
## show case
![](./showcase.gif)
## how to install
1. install fzf,zplug.
2. 
```bash
zplug "woodgear/zmx"

# load all shell-actions
export SHELL_ACTIONS_PATH=$DIR_CONTAINER_YOU_ACTION_SCRIPT
# you could just add you script
export SHELL_ACTIONS_PATH=$SHELL_ACTIONS_PATH:$YOUR_SCRIPT_DIR

load_shell_actions $SHELL_ACTIONS_PATH
```
## how to use
1. dealut bindkey is  `,xm`.
2. action come from script which file path in SHELL_ACTIONS_PATH.
## annotation
1. @arg-len action(function) which has comment like '# @arg-len:1' will not eval but wait you args.