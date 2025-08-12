- Add the ability to set "modes": "plan" and "execute". In headless mode, "execute" is the default mode unless the "--plan" flag is passed. In TUI mode the default mode is "execute" and can be changed by pressing <shift>+<tab>

- Lets create a simple system for canceling in progress actions. We should be able to leverage something in the langchain-go library already for this. We should do some research to see what the best approach might be here and create a comprehensive plan for adding.

- Doublecheck our tools are following the way that langchain expects these: https://tmc.github.io/langchaingo/docs/modules/agents/ and for the chains as well

- Text processing middleware and markdown styling.
- text splitters for large text input
- memory adjustments: https://tmc.github.io/langchaingo/docs/modules/memory/ need to adjust the mode dynamically
