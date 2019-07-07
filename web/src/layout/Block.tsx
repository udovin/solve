import React, {ReactNode} from "react";

interface BlockProps {
	header?: ReactNode,
	footer?: ReactNode,
	title?: string,
}

export default class Block extends React.Component<BlockProps> {
	render(): ReactNode {
		const header = this.renderHeader();
		const content = this.renderContent();
		const footer = this.renderFooter()
		return (
			<div className="ui-block-wrap">
				<div className="ui-block">{header}{content}{footer}</div>
			</div>
		);
	}

	renderHeader(): ReactNode {
		const {title} = this.props;
		if (title) {
			return (
				<div className="ui-block-header">
					<span className="title">{title}</span>
				</div>
			);
		}
	}

	renderContent(): ReactNode {
		return (
			<div className="ui-block-content">{this.props.children}</div>
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
