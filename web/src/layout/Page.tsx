import React, {ReactNode} from "react";

export default class Page extends React.Component<any> {
	componentWillMount(): void {
		document.title = this.props.title
	}

	render(): ReactNode {
		const sidebar = this.renderSidebar();
		const content = this.renderContent();
		return (
			<main id="main" className={this.props.sidebar ? "" : ""}>
				{sidebar}
				{content}
			</main>
		);
	}

	renderSidebar(): ReactNode {
		const {sidebar} = this.props;
		if (!sidebar) {
			return null
		}
		return (
			<div id="sidebar-wrap">
				<div id="sidebar">{sidebar}</div>
			</div>
		);
	}

	renderContent(): ReactNode {
		const {children} = this.props;
		return (
			<div id="content-wrap">
				<div id="content">{children}</div>
			</div>
		);
	}
}
