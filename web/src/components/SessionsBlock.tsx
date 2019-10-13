import React, {FC} from "react";
import Block, {BlockProps} from "./Block";
import {Session} from "../api";
import "./SessionsBlock.scss";

export type SessionsBlockProps = BlockProps & {
	sessions: Session[];
	currentSession?: Session;
};

const SessionsBlock: FC<SessionsBlockProps> = props => {
	const {sessions, currentSession, ...rest} = props;
	const onClick = (session: Session) => {
		const {ID} = session;
		fetch(`/api/v0/sessions/${ID}`, {method: "DELETE"})
			.then(result => result.json());
	};
	return <Block className="b-sessions" title="Sessions" {...rest}>
		<table className="ui-table">
			<thead>
			<tr>
				<th className="id">#</th>
				<th className="actions">Actions</th>
			</tr>
			</thead>
			<tbody>
			{sessions && sessions.map((session, key) => {
				const {ID} = session;
				return <tr key={key} className="session">
					<td className="id">{ID}</td>
					<td className="actions">
						{(!currentSession || ID !== currentSession.ID) &&
						<button onClick={() => onClick(session)}>Close</button>}
					</td>
				</tr>;
			})}
			</tbody>
		</table>
	</Block>
};

export default SessionsBlock;
