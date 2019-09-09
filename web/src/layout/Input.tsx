import React, {FC, InputHTMLAttributes} from "react";

export type InputProps = InputHTMLAttributes<HTMLInputElement>

const Input: FC<InputProps> = props => {
	const {...rest} = props;
	return <input className="ui-input" {...rest}/>
};

export default Input;
