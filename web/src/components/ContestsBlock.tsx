import React, {FC} from "react";
import {Link} from "react-router-dom";
import Block, {BlockProps} from "../ui/Block";
import {Contest} from "../api";
import "./ContestsBlock.scss"

export type ContestsBlockProps = BlockProps & {
	contests: Contest[];
};

const ContestsBlock: FC<ContestsBlockProps> = props => {
	const {contests, ...rest} = props;
	return <Block className="b-contests" title="Contests" {...rest}>
		<table className="ui-table">
			<thead>
			<tr>
				<th className="title">Title</th>
			</tr>
			</thead>
			<tbody>
			{contests && contests.map((contest, index) => {
				const {ID, Title} = contest;
				return <tr key={index} className="contest">
					<td className="title">
						<Link to={`/contests/${ID}`}>{Title}</Link>
					</td>
				</tr>;
			})}
			</tbody>
		</table>
	</Block>
};

export default ContestsBlock;
