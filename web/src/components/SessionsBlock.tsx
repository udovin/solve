import React, {FC} from "react";
import Block, {BlockProps} from "../ui/Block";
import {Session} from "../api";
import "./SessionsBlock.scss";

export type SessionsBlockProps = BlockProps & {
	sessions: Session[];
	currentSession?: Session;
};

const SessionsBlock: FC<SessionsBlockProps> = props => {
	const {sessions, currentSession, ...rest} = props;
	const onClick = (session: Session) => {
		const {id} = session;
		fetch(`/api/v0/sessions/${id}`, {method: "DELETE"})
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
				const {id} = session;
				return <tr key={key} className="session">
					<td className="id">{id}</td>
					<td className="actions">
						{(!currentSession || id !== currentSession.id) &&
						<button onClick={() => onClick(session)}>Close</button>}
					</td>
				</tr>;
			})}
			</tbody>
		</table>
	</Block>
};

export default SessionsBlock;
