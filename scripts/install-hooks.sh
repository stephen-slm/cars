#!/bin/bash

HOOK_NAMES="applypatch-msg pre-applypatch post-applypatch pre-commit prepare-commit-msg commit-msg post-commit pre-rebase post-checkout post-merge pre-receive update post-receive post-update pre-auto-gc"

# relative folder path of the .git hook based on the execution place of the script.
GIT_HOOK_DIR=./.git/hooks
# relative folder path of the custom hooks based on the execution place of the script.
LOCAL_HOOK_DIR=./scripts/githooks
# relative folder path of the custom hooks to deploy based on the .git hook folder
LNS_RELATIVE_PATH=../../scripts/githooks

echo "Install project git hooks"

for hook in $HOOK_NAMES; do
    # if we have a custom hook to set
    if [ -f $LOCAL_HOOK_DIR/$hook ]; then
      echo "> Hook $hook"
      # If the hook already exists, is executable, and is not a symlink
      if [ ! -h $GIT_HOOK_DIR/$hook -a -x $GIT_HOOK_DIR/$hook ]; then
          echo " > Old git hook $hook disabled"
          # append .local to disable it
          mv $GIT_HOOK_DIR/$hook $GIT_HOOK_DIR/$hook.local
      fi

      # create the symlink, overwriting the file if it exists
      echo " > Enable project git hook"
      ln -s -f $LNS_RELATIVE_PATH/$hook $GIT_HOOK_DIR/$hook
    fi
done

