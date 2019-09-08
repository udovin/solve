import React, {FC, FormEventHandler, ReactNode} from "react";

type BlockProps = {
	title?: string;
	header?: ReactNode;
	footer?: ReactNode;
};

export const Block: FC<BlockProps> = ({title, header, footer, children}) => {
	if (title) {
		header = <span className="title">{title}</span>;
	}
	return <div className="ui-block-wrap">
		<div className="ui-block">
			{header && <div className="ui-block-header">{header}</div>}
			<div className="ui-block-content">{children}</div>
			{footer && <div className="ui-block-footer">{footer}</div>}
		</div>
	</div>;
};

type FormBlockProps = BlockProps & {
	onSubmit: FormEventHandler<HTMLFormElement>;
};

export const FormBlock: FC<FormBlockProps> = ({title, header, footer, onSubmit, children}) => {
	if (title) {
		header = <span className="title">{title}</span>;
	}
	return <div className="ui-block-wrap">
		<form className="ui-block" onSubmit={onSubmit}>
			{header && <div className="ui-block-header">{header}</div>}
			<div className="ui-block-content">{children}</div>
			{footer && <div className="ui-block-footer">{footer}</div>}
		</form>
	</div>;
};
