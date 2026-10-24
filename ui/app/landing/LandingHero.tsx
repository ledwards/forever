import i18n from '@i18n/config';
import { Trans } from 'react-i18next';

import { ReleaseInfo } from '../ReleaseInfo';
import { LandingActions } from './LandingActions';

export const LandingHero = () => (
	<section className="mb-12 flex max-w-landing-hero flex-col">
		<div className="flex flex-wrap items-baseline gap-x-6">
			<h1 className="ui-landing-hero-title" data-testid="landing-heading">
				<Trans i18n={i18n} i18nKey="landing.home.heading" components={{ expansion: <span className="text-expansion" /> }} />
			</h1>
			<ReleaseInfo className="text-sm text-white/65 motion-safe:animate-landing-rise-in-slow" />
		</div>
		<p id="description" className="ui-landing-hero-blurb">
			<Trans
				i18n={i18n}
				i18nKey="landing.home.welcomeDescription"
				components={{
					brand: <strong className="text-brand" />,
					bold: <strong />,
					expansion: <strong className="text-expansion" />,
				}}
			/>
		</p>
		<LandingActions />
	</section>
);
