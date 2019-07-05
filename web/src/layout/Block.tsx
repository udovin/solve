import React, {ReactNode} from "react";

export class Block extends React.Component<{title?: string, footer?: any}> {
	render(): ReactNode {
		return (
			<div className="ui-block">
				{this.renderHeader()}
				<div className="ui-block-content">{this.props.children}</div>
				{this.renderFooter()}
			</div>
		);
	}

	renderHeader(): ReactNode {
		return (
			<div className="ui-block-header">
				<span className="title">{this.props.title}</span>
			</div>
		);
	}

	renderFooter(): ReactNode {
		if (this.props.footer) {
			return (
				<div className="ui-block-footer">{this.props.footer}</div>
			);
		}
		return null;
	}
}
