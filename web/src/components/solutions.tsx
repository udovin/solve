import React, {FC, FormEventHandler} from "react";
import Block, {BlockProps} from "../ui/Block";
import Button from "../ui/Button";
import {Compiler, getDefense, getShortVerdict, Solution} from "../api";
import Input from "../ui/Input";
import {Link} from "react-router-dom";
import "./solutions.scss";
import FormBlock from "./FormBlock";
import SolutionRow from "./SolutionRow";
import Textarea from "../ui/Textarea";

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
				<select className="ui-select" name="compilerID">
					{compilers && compilers.map((compiler, index) =>
						<option value={compiler.ID} key={index}>{compiler.Name}</option>
					)}
				</select>
			</label>
			<label>
				<span className="label">Source file:</span>
				<Input type="file" name="sourceFile" placeholder="Source code"/>
			</label>
			<label>
				<span className="label">Source text:</span>
				<Textarea name="sourceText" placeholder="Source text"/>
			</label>
		</div>
	</FormBlock>;
};

export type SolutionsBlockProps = BlockProps & {
	solutions: Solution[];
};

export const SolutionsBlock: FC<SolutionsBlockProps> = props => {
	let {solutions, className, ...rest} = props;
	className = className ? "b-solutions " + className : "b-solutions";
	return <Block className={className} {...rest}>
		<table className="ui-table">
			<thead>
			<tr>
				<th className="id">#</th>
				<th className="created">Created</th>
				<th className="participant">Participant</th>
				<th className="problem">Problem</th>
				<th className="verdict">Verdict</th>
			</tr>
			</thead>
			<tbody>
			{solutions && solutions.map((solution, key) =>
				<SolutionRow showID={true} solution={solution} key={key}/>
			)}
			</tbody>
		</table>
	</Block>;
};

export const SolutionsSideBlock: FC<SolutionsBlockProps> = props => {
	let {solutions, className, ...rest} = props;
	className = className ? "b-solutions " + className : "b-solutions";
	const format = (n: number) => {
		return ("0" + n).slice(-Math.max(2, String(n).length));
	};
	const formatDate = (d: Date) =>
		[d.getFullYear(), d.getMonth() + 1, d.getDate()].map(format).join("-");
	const formatTime = (d: Date) =>
		[d.getHours(), d.getMinutes(), d.getSeconds()].map(format).join(":");
	return <Block className={className} {...rest}>
		<table className="ui-table">
			<thead>
			<tr>
				<th className="id">#</th>
				<th className="created">Created</th>
				<th className="verdict">Verdict</th>
			</tr>
			</thead>
			<tbody>
			{solutions && solutions.map((solution, index) => {
				const {ID, CreateTime, Report} = solution;
				let createDate = new Date(CreateTime * 1000);
				return <tr key={index} className="solution">
					<td className="id">
						<Link to={"/solutions/" + ID}>{ID}</Link>
					</td>
					<td className="created">
						<div className="time">{formatTime(createDate)}</div>
						<div className="date">{formatDate(createDate)}</div>
					</td>
					<td className="verdict">
						<div className="type">{Report && getShortVerdict(Report.Verdict)}</div>
						<div className="value">{Report && Report.Data.Points}</div>
						<div className="defense">{Report && getDefense(Report.Data.Defense)}</div>
					</td>
				</tr>;
			})}
			</tbody>
		</table>
	</Block>;
};
