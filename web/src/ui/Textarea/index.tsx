import React, {FC, TextareaHTMLAttributes} from "react";
import "./index.scss";

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

const Index: FC<TextareaProps> = props => {
	const {...rest} = props;
	return <textarea className="ui-textarea" {...rest}/>
};

export default Index;
