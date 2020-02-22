import React, {FC, InputHTMLAttributes} from "react";
import "./index.scss";

export type InputProps = InputHTMLAttributes<HTMLInputElement>;

const Input: FC<InputProps> = props => {
	const {...rest} = props;
	return <input className="ui-input" {...rest}/>
};

export default Input;
