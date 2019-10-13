import React, {Attributes, FC} from "react";
import {Link} from "react-router-dom";
import {getDefense, getShortVerdict, Solution} from "../api";

export type SolutionRowProps = Attributes & {
	showID: boolean;
	solution: Solution;
};

const SolutionRow: FC<SolutionRowProps> = props => {
	const format = (n: number) => {
		return ("0" + n).slice(-Math.max(2, String(n).length));
	};
	const formatDate = (d: Date) =>
		[d.getFullYear(), d.getMonth() + 1, d.getDate()].map(format).join("-");
	const formatTime = (d: Date) =>
		[d.getHours(), d.getMinutes(), d.getSeconds()].map(format).join(":");
	const {showID, solution, key} = props;
	const {ID, CreateTime, User, Problem, Report} = solution;
	let createDate = new Date(CreateTime * 1000);
	return <tr key={key} className="solution">
		{showID && <td className="id"><Link to={`/solutions/${ID}`}>{ID}</Link></td>}
		<td className="created">
			<div className="time">{formatTime(createDate)}</div>
			<div className="date">{formatDate(createDate)}</div>
		</td>
		<td className="participant">{User ?
			<Link to={`/users/${User.Login}`}>{User.Login}</Link> :
			<>&mdash;</>
		}</td>
		<td className="problem">{Problem ?
			<Link to={`/problems/${Problem.ID}`}>{Problem.Title}</Link> :
			<span>&mdash;</span>
		}</td>
		<td className="verdict">
			<div className="type">{Report && getShortVerdict(Report.Verdict)}</div>
			<div className="value">{Report && Report.Data.Points}</div>
			<div className="defense">{Report && getDefense(Report.Data.Defense)}</div>
		</td>
	</tr>;
};

export default SolutionRow;
