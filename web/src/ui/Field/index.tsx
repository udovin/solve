import React, {BaseHTMLAttributes, FC, ReactNode} from "react";
import "./index.scss";

export type FieldProps = BaseHTMLAttributes<HTMLDivElement> & {
	title?: string;
	description?: string | ReactNode;
};

const Field: FC<FieldProps> = props => {
	const {title, description, children, ...rest} = props;
	return <div className="ui-field" {...rest}>
		<label>
			{title && <span className="label">{title}</span>}
			{children}
		</label>
		{description && <span className="text">{description}</span>}
	</div>;
};

export default Field;
