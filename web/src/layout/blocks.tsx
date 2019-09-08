import React, {
	BaseHTMLAttributes,
	FC,
	FormHTMLAttributes,
	ReactNode
} from "react";

type BlockProps = BaseHTMLAttributes<HTMLDivElement> & {
	title?: string;
	header?: ReactNode;
	footer?: ReactNode;
};

export const Block: FC<BlockProps> = props => {
	let {title, header, footer, children, ...rest} = props;
	if (title) {
		header = <span className="title">{title}</span>;
	}
	return <div className="ui-block-wrap">
		<div className="ui-block" {...rest}>
			{header && <div className="ui-block-header">{header}</div>}
			<div className="ui-block-content">{children}</div>
			{footer && <div className="ui-block-footer">{footer}</div>}
		</div>
	</div>;
};

type FormBlockProps = BlockProps & FormHTMLAttributes<HTMLFormElement>;

export const FormBlock: FC<FormBlockProps> = props => {
	let {title, header, footer, children, ...rest} = props;
	if (title) {
		header = <span className="title">{title}</span>;
	}
	return <div className="ui-block-wrap">
		<form className="ui-block" {...rest}>
			{header && <div className="ui-block-header">{header}</div>}
			<div className="ui-block-content">{children}</div>
			{footer && <div className="ui-block-footer">{footer}</div>}
		</form>
	</div>;
};
