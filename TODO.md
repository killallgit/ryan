# CLEANUP
- Do a full review of the code and find any interfaces that might be duplicated, any dead code, or code that can be logically combined with existing so interfaces and logic is unified.
- Many of our package files are very long. Lets identify where these can be split up for readibility. Packages and files should be "single purpose" and easy to test.
- Cleanup the .ryan settings dir. We now need a .ryan/logs/<log related files>. We now also need a .ryan/contexts dir where we can persist raw context that langchain is using. We need to make sure we're in line with how langchain handles this and use their primitives for managing context. If langchain does not have a built in way to persist context to file, we can reasses this
- We need to have a record of the chat history. Lets store a file .ryan/logs/debug.history that is always overwritten the next time the app is ran.
- cleanup all the test commands. Put into their own namespace and optimize them


# CONTEXT
- Lets being integrating more full featured context. We will start by storing our context to disk. Be
