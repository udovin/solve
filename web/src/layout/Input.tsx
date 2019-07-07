import React, {ReactNode, InputHTMLAttributes} from "react";

interface Props extends InputHTMLAttributes<HTMLInputElement> {}

export default class Input extends React.Component<Props> {
	render(): ReactNode {
		const {...rest} = this.props;
		return (
			<input className="ui-input" {...rest}/>
		);
	}
}
