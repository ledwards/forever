import type { Meta, StoryObj } from '@storybook/react-vite';
import { RELEASE, withRelease } from '@testing/build_info';

import { ReleaseInfo } from './ReleaseInfo';

const meta = {
	title: 'App/ReleaseInfo',
	component: ReleaseInfo,
	tags: ['autodocs'],
	parameters: { spec: 'none' },
	args: { className: 'text-sm text-white/65' },
	beforeEach: ({ parameters }) => withRelease(parameters.release ?? RELEASE),
} satisfies Meta<typeof ReleaseInfo>;

export default meta;
type Story = StoryObj<typeof meta>;

/** The landing page's line, beside the title. */
export const Default: Story = {};

// A build that could not read its commit, such as one made from a source archive.
export const WithoutCommit: Story = {
	parameters: { release: { ...RELEASE, commit: '' } },
};
