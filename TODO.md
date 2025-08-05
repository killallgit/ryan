- Move the ctrl-t tree context modal into its own separate view that can be accessed in the command pallete
- add another command pallete view for tools. I want to see which are registered, how many times they've been called, if they're running, and any other info we can fit into a simple but nice view. 
- It would be great if we could somehow test up front that the selected model can actually use each tool by testing it actually can. I would want to put this into it's own background process as to not block the thread.
- make the command pallet an input field that can filter on typing or items can be selected by pressing <tab>


1. On startup we register each agent-expert from the yaml def
1. Take prompt and pre-process / plan
2. Make a list of potential tools to use
