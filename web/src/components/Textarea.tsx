import React, {FC, TextareaHTMLAttributes} from "react";

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

const Textarea: FC<TextareaProps> = props => {
	const {...rest} = props;
	return <textarea className="ui-textarea" {...rest}/>
};

export default Textarea;
