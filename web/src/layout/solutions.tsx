import React, {FC, FormEventHandler} from "react";
import {FormBlock} from "./blocks";
import {Button} from "./buttons";
import {Compiler} from "../api";

type SubmitSolutionSideBlockProps = {
	onSubmit: FormEventHandler;
	compilers?: Compiler[];
};

const SubmitSolutionSideBlock: FC<SubmitSolutionSideBlockProps> = props => {
	const {onSubmit, compilers} = props;
	return <FormBlock onSubmit={onSubmit} title="Submit solution" footer={
		<Button color="primary">Submit</Button>
	}>
		<div className="ui-field">
			<label>
				<span className="label">Compiler:</span>
				<select name="compilerID">
					{compilers && compilers.map((compiler, index) =>
						<option value={compiler.ID} key={index}>{compiler.Name}</option>
					)}
				</select>
			</label>
			<label>
				<span className="label">Source file:</span>
				<input type="file" name="sourceFile" placeholder="Source code"/>
			</label>
		</div>
	</FormBlock>;
};

export {SubmitSolutionSideBlock};
