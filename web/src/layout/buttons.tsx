import React, {ButtonHTMLAttributes, FunctionComponent} from "react";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

export const Button: FunctionComponent<ButtonProps> = (props) => {
	const {color, ...rest} = props;
	return <button className={["ui-button", color].join(" ")} {...rest}/>;
};
