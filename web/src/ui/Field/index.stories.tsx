import React from "react";
import Field from "./index";
import "../../index.scss";
import Input from "../Input";

export default {title: "Field"};

export const common = () => <>
	<Field title="Title" description="Description">
		<Input defaultValue="Input"/>
	</Field>
</>;
