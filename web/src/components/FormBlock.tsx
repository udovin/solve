import React, {FC, FormHTMLAttributes} from "react";
import {BlockProps} from "../ui/Block";

export type FormBlockProps = BlockProps & FormHTMLAttributes<HTMLFormElement>;

const FormBlock: FC<FormBlockProps> = props => {
	let {title, header, footer, children, className, ...rest} = props;
	if (title) {
		header = <span className="title">{title}</span>;
	}
	className = className ? "ui-block-wrap " + className : "ui-block-wrap";
	return <div className={className} {...rest}>
		<form className="ui-block">
			{header && <div className="ui-block-header">{header}</div>}
			<div className="ui-block-content">{children}</div>
			{footer && <div className="ui-block-footer">{footer}</div>}
		</form>
	</div>;
};

export default FormBlock;
