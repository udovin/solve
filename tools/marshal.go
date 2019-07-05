package tools

type InterfaceUnmarshal []byte

func (g *InterfaceUnmarshal) UnmarshalJSON(bytes []byte) error {
	*g = bytes
	return nil
}
