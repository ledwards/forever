import i18n from '@i18n/config';
import { commitUrl } from '@sim/constants/build_info';
import { Button } from '@ui-kit/Button';
import type { ReactNode } from 'react';

/** A commit's short hash, linked to the commit on GitHub. */
export const CommitLink = ({ commit }: { commit: string }) => {
	const short = commit.slice(0, 7);
	return (
		<Button href={commitUrl(commit)} target="_blank" variant="unstyled" title={i18n.t('common.release_info.commit_link', { commit: short })}>
			{short}
		</Button>
	);
};

/** The parts given, empty ones left out, separated by ` · `. */
export const joinParts = (parts: ReactNode[]): ReactNode[] => parts.filter(Boolean).flatMap((part, index) => (index > 0 ? [' · ', part] : [part]));
