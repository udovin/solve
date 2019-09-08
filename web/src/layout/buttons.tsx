import React, {ButtonHTMLAttributes, FC} from "react";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

export const Button: FC<ButtonProps> = (props) => {
	const {color, ...rest} = props;
	return <button className={["ui-button", color].join(" ")} {...rest}/>;
};
