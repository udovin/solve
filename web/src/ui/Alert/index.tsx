import React, {BaseHTMLAttributes, FC} from "react";
import "./index.scss";

export enum AlertKind {
	SUCCESS = "success",
	INFO = "info",
	WARNING = "warning",
	DANGER = "danger",
}

export type AlertProps = BaseHTMLAttributes<HTMLDivElement> & {
	kind?: AlertKind;
};

const Alert: FC<AlertProps> = props => {
	const {kind, children, ...rest} = props;
	return <div className={"ui-alert " + (kind || AlertKind.DANGER)} {...rest}>
		<div className="ui-alert-content">{children}</div>
	</div>;
};

export default Alert;
