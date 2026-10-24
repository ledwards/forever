import { PlayerSpecs } from '@sim/player/specs/index';
import type { Meta, StoryObj } from '@storybook/react-vite';
import { RELEASE, withRelease } from '@testing/build_info';

import { SpecReleaseInfo } from './SpecReleaseInfo';

const meta = {
	title: 'App/SpecReleaseInfo',
	component: SpecReleaseInfo,
	tags: ['autodocs'],
	parameters: { spec: 'none' },
	args: {
		spec: PlayerSpecs.FeralCatDruid,
		className: 'border-t border-border px-sidebar py-2 text-xs text-muted',
	},
	beforeEach: () => withRelease(RELEASE),
} satisfies Meta<typeof SpecReleaseInfo>;

export default meta;
type Story = StoryObj<typeof meta>;

/** The sidebar's line; its info button opens the sim build and the spec's last preset and engine changes. */
export const Default: Story = {};

// A build from a shallow clone reads no change for the spec, so the details list only the sim build.
export const WithoutChanges: Story = {
	args: { spec: PlayerSpecs.DpsWarrior },
};
