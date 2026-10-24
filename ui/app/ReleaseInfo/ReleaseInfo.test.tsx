import i18n from '@i18n/config';
import { RELEASE, withRelease } from '@testing/build_info';
import { render, screen } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

import english from '../../../assets/locales/en/translation.json';
import { ReleaseInfo } from './ReleaseInfo';

beforeAll(() => {
	i18n.addResourceBundle('en', 'translation', english, true, true);
});
afterAll(() => {
	i18n.removeResourceBundle('en', 'translation');
});

let restore = () => {};
afterEach(() => restore());

describe('ReleaseInfo', () => {
	it('shows the game build, the build date and the short commit, linked to GitHub', () => {
		restore = withRelease(RELEASE);
		render(<ReleaseInfo />);
		expect(screen.getByRole('paragraph').textContent).toBe('Game 1.60.1.70450 · Built 2026-10-18 · 2fe052c');
		const link = screen.getByRole('link', { name: '2fe052c' });
		expect(link.getAttribute('href')).toBe(`https://github.com/wowsims/forever/commit/${RELEASE.commit}`);
		expect(link.getAttribute('rel')).toBe('noopener noreferrer');
	});

	it('names the game build it is given', () => {
		restore = withRelease(RELEASE);
		render(<ReleaseInfo gameVersion="1.60.1.70425" />);
		expect(screen.getByRole('paragraph').textContent).toBe('Game 1.60.1.70425 · Built 2026-10-18 · 2fe052c');
	});

	it('leaves out what the build could not read', () => {
		restore = withRelease({ ...RELEASE, commit: '', gameVersion: '' });
		render(<ReleaseInfo />);
		expect(screen.getByRole('paragraph').textContent).toBe('Built 2026-10-18');
		expect(screen.queryByRole('link')).toBeNull();
	});
});
