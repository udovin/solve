import React, {BaseHTMLAttributes, FC, ReactNode} from "react";

export type BlockProps = BaseHTMLAttributes<HTMLDivElement> & {
	title?: string;
	header?: ReactNode;
	footer?: ReactNode;
};

const Block: FC<BlockProps> = props => {
	let {title, header, footer, children, className, ...rest} = props;
	if (title) {
		header = <span className="title">{title}</span>;
	}
	className = className ? "ui-block-wrap " + className : "ui-block-wrap";
	return <div className={className} {...rest}>
		<div className="ui-block">
			{header && <div className="ui-block-header">{header}</div>}
			<div className="ui-block-content">{children}</div>
			{footer && <div className="ui-block-footer">{footer}</div>}
		</div>
	</div>;
};

export default Block;
