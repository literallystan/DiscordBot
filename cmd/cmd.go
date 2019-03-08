package cmd

//Command ...
type Command struct {
	cmds map[string]func()
}

//NewCommand creates a map of string to func
/* func NewCommand() *Command {
	return &Command(_, make())
} */

//GetCmds ...
/* func (handler Command) GetCmds() CmdMap {
	return handler.cmds
}

//Get ...
func (handler Command) Get(name string) (*Action, bool) {
	cmd, found := handler.cmds[name]
	return &cmd, found
}

func (handler Command) Register(name string, command Command) {
	handler.cmds[name] = command
	if len(name) > 1 {
		handler.cmds[name[:1]] = command
	}
}
*/
