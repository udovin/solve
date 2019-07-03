import React, {ReactNode} from "react";

export class Block extends React.Component<any> {
	render(): ReactNode {
		return (
			<div className="ui-block">
				<div className="ui-block-header">
					<span className="title">{this.props.title}</span>
				</div>
				<div className="ui-block-content">{this.props.children}</div>
				<div className="ui-block-footer">{this.props.footer}</div>
			</div>
		);
	}
}
