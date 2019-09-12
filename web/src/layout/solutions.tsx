import React, {FC, FormEventHandler} from "react";
import {Block, FormBlock} from "./blocks";
import {Button} from "./buttons";
import {Compiler, getShortVerdict, Solution} from "../api";
import Input from "./Input";
import {Link} from "react-router-dom";

export type SubmitSolutionSideBlockProps = {
	onSubmit: FormEventHandler;
	compilers?: Compiler[];
};

export const SubmitSolutionSideBlock: FC<SubmitSolutionSideBlockProps> = props => {
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
				<Input type="file" name="sourceFile" placeholder="Source code"/>
			</label>
		</div>
	</FormBlock>;
};

export type SolutionsSideBlockProps = {
	solutions: Solution[];
};

export const SolutionsSideBlock: FC<SolutionsSideBlockProps> = props => {
	const {solutions} = props;
	return <Block title="Solutions">
		<ul>{solutions && solutions.map(
			(solution, index) => <li key={index}>
				<Link to={"/solutions/" + solution.ID}>{solution.ID}</Link>
				{solution.Report && <span className="verdict">{getShortVerdict(solution.Report.Verdict)}</span>}
			</li>
		)}</ul>
	</Block>
};
