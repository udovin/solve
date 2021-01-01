import React from "react";
import Input from "./index";

const config = {title: "Input"};

export default config;

export const common = () => <>
	<p><Input defaultValue="Default"/></p>
	<p><Input defaultValue="Disabled" disabled/></p>
</>;
