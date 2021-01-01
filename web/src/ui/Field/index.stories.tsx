import React from "react";
import Field from "./index";
import "../../index.scss";
import Input from "../Input";

const config = {title: "Field"};

export default config;

export const common = () => <>
	<Field title="Title" description="Description">
		<Input defaultValue="Input"/>
	</Field>
</>;
