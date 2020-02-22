import React, {ButtonHTMLAttributes, FC} from "react";
import "./index.scss";

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement>;

const Button: FC<ButtonProps> = (props) => {
	const {color, ...rest} = props;
	return <button className={["ui-button", color].join(" ")} {...rest}/>;
};

export default Button;
