import React from "react";
import Textarea from "./index";

const config = {title: "Textarea"};

export default config;

export const common = () => <>
	<p><Textarea defaultValue="Default"/></p>
	<p><Textarea defaultValue="Disabled" disabled/></p>
</>;
