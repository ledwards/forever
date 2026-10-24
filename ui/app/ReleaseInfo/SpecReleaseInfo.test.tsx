import i18n from '@i18n/config';
import { translatePlayerClass, translatePlayerSpec } from '@i18n/localization';
import { PlayerSpecs } from '@sim/player/specs/index';
import { RELEASE, withRelease } from '@testing/build_info';
import { fireEvent, render, screen } from '@testing-library/react';
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

import character from '../../../assets/locales/en/character.json';
import english from '../../../assets/locales/en/translation.json';
import { SpecReleaseInfo } from './SpecReleaseInfo';

beforeAll(() => {
	i18n.addResourceBundle('en', 'translation', english, true, true);
	i18n.addResourceBundle('en', 'character', character, true, true);
});
afterAll(() => {
	i18n.removeResourceBundle('en', 'translation');
	i18n.removeResourceBundle('en', 'character');
});

let restore = () => {};
afterEach(() => restore());

const details = () => {
	fireEvent.click(screen.getByRole('button', { name: 'Release details' }));
	return {
		labels: screen.getAllByRole('term').map(term => term.textContent),
		values: screen.getAllByRole('definition').map(definition => definition.textContent),
	};
};

describe('SpecReleaseInfo', () => {
	it("names the game build of the spec's own spell data and the build date, leaving the commit to the details", () => {
		restore = withRelease(RELEASE);
		render(<SpecReleaseInfo spec={PlayerSpecs.FeralCatDruid} />);
		expect(screen.getByRole('paragraph').textContent).toBe('Game 1.60.1.70425 · Built 2026-10-18');
		expect(screen.queryByRole('link')).toBeNull();
	});

	it('lists the sim build and the last preset and engine changes on clicking its info button', () => {
		restore = withRelease(RELEASE);
		render(<SpecReleaseInfo spec={PlayerSpecs.FeralCatDruid} />);
		expect(screen.queryByRole('definition')).toBeNull();

		const { labels, values } = details();
		const spec = translatePlayerSpec(PlayerSpecs.FeralCatDruid);
		expect(labels).toEqual(['Sim build', `${spec} presets`, `${spec} engine`]);
		expect(values).toEqual(['Built 2026-10-18 · 2fe052c', 'Changed 2026-10-17 · 8109a4c', 'Changed 2026-10-16 · 474c498']);
		const engine = RELEASE.specs['druid/feralcat'].engine!.commit;
		expect(screen.getByRole('link', { name: '474c498' }).getAttribute('href')).toBe(`https://github.com/wowsims/forever/commit/${engine}`);
	});

	it('names the rows of a spec without a name of its own after its class', () => {
		restore = withRelease(RELEASE);
		render(<SpecReleaseInfo spec={PlayerSpecs.Mage} />);
		expect(details().labels).toEqual(['Sim build', `${translatePlayerClass(PlayerSpecs.getPlayerClass(PlayerSpecs.Mage))} presets`]);
	});

	it('lists only the sim build when the build read no change for the spec', () => {
		restore = withRelease(RELEASE);
		render(<SpecReleaseInfo spec={PlayerSpecs.DpsWarrior} />);
		expect(details().labels).toEqual(['Sim build']);
	});
});
