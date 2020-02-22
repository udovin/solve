import React from "react";
import Input from "./index";

export default {title: "Input"};

export const common = () => <>
	<p><Input defaultValue="Default"/></p>
	<p><Input defaultValue="Disabled" disabled/></p>
</>;
