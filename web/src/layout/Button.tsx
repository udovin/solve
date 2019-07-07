import React, {ReactNode, ButtonHTMLAttributes} from "react";

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {}

export default class Button extends React.Component<Props> {
	render(): ReactNode {
		const {color, ...rest} = this.props;
		return (
			<button className={["ui-button", color].join(" ")} {...rest}/>
		);
	}
}
